<?xml version="1.0" encoding="UTF-8"?>


<!-- 
        * This is a stylesheet for converting XML to JSON. 
        * It expects the XML to be in the format produced by the XSLT 3.0 function
        * fn:json-to-xml(), but is designed to be highly customizable.
        *
        * The stylesheet is made available under the terms of the W3C software notice and license
        * at http://www.w3.org/Consortium/Legal/copyright-software-19980720
        *
    -->    

<xsl:package
    name="http://www.w3.org/2013/XSLT/xml-to-json"
    package-version="1.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:fn="http://www.w3.org/2005/xpath-functions"
    xmlns:j="http://www.w3.org/2013/XSLT/xml-to-json"
    exclude-result-prefixes="xs fn j" default-mode="j:xml-to-json" version="3.0">
    
    <xsl:variable name="quot" visibility="private">"</xsl:variable>
    <xsl:param name="indent-spaces" select="2"/>
    
    <!-- The static parameter STREAMABLE controls whether the stylesheet is declared as streamable -->
    
    <xsl:param name="STREAMABLE" static="yes" as="xs:boolean" select="true()"/>
    
    <xsl:mode name="j:xml-to-json"/>  <!-- see spec bug 30288 -->
    <xsl:mode name="indent" _streamable="{$STREAMABLE}" visibility="public"/>
    <xsl:mode name="no-indent" _streamable="{$STREAMABLE}" visibility="public"/>
    <xsl:mode name="key-attribute" streamable="false" on-no-match="fail" visibility="public"/>
    
    <!-- The static parameter VALIDATE controls whether the input, if untyped, should be validated -->
    
    <xsl:param name="VALIDATE" static="yes" as="xs:boolean" select="false()"/>
    <xsl:import-schema namespace="http://www.w3.org/2005/xpath-functions" use-when="$VALIDATE"/>
    
    <!-- Entry point: function to convert a supplied XML node to a JSON string -->
    <xsl:function name="j:xml-to-json" as="xs:string" visibility="public" 
                _streamability="{if ($STREAMABLE) then 'absorbing' else 'unclassified'}">
        <xsl:param name="input" as="node()"/>
        <xsl:sequence select="j:xml-to-json($input, map{})"/>
    </xsl:function>
    
    <!-- Entry point: function to convert a supplied XML node to a JSON string, supplying options -->
    <xsl:function name="j:xml-to-json" as="xs:string" visibility="public" 
                _streamability="{if ($STREAMABLE) then 'absorbing' else 'unclassified'}">
        <xsl:param name="input" as="node()"/>
        <xsl:param name="options" as="map(*)"/>
        <xsl:variable name="input" as="node()" use-when="$VALIDATE">
            <xsl:copy-of select="$input" validation="strict"/>
        </xsl:variable>
        <xsl:choose>
            <xsl:when test="$options('indent') eq true()">
                <xsl:apply-templates select="$input" mode="indent">
                    <xsl:with-param name="fallback" as="(function(element()) as xs:string)?"
                        select="$options('fallback')" tunnel="yes"/>
                </xsl:apply-templates>
            </xsl:when>
            <xsl:otherwise>
                <xsl:apply-templates select="$input" mode="no-indent">
                    <xsl:with-param name="fallback" as="(function(element()) as xs:string)?"
                        select="$options('fallback')" tunnel="yes"/>
                </xsl:apply-templates>
            </xsl:otherwise>
        </xsl:choose>
    </xsl:function>
    
    <!-- A document node is ignored -->
    
    <xsl:template match="/" mode="indent no-indent">
        <xsl:apply-templates mode="#current"/>
    </xsl:template>
    
    <!-- Template rule for fn:map elements, representing JSON objects -->
    
    <xsl:template match="fn:map" mode="indent">
        <xsl:value-of>
            <xsl:variable name="depth" select="count(ancestor::*) + 1"/>
            <xsl:text>{</xsl:text>
            <xsl:for-each select="*">
                <xsl:if test="position() gt 1">
                    <xsl:text>, </xsl:text>
                    <xsl:value-of select="j:indent($depth)"/>
                </xsl:if>
                <xsl:apply-templates select="snapshot(@key)" mode="key-attribute"/>
                <xsl:text> : </xsl:text>
                <xsl:apply-templates select="." mode="#current"/>
            </xsl:for-each>
            <xsl:text>}</xsl:text>
        </xsl:value-of>
    </xsl:template>
    
    <xsl:template match="fn:map" mode="no-indent">
        <xsl:value-of>
            <xsl:text>{</xsl:text>
            <xsl:for-each select="*">
                <xsl:if test="position() gt 1">
                    <xsl:text>,</xsl:text>
                </xsl:if>
                <xsl:apply-templates select="snapshot(@key)" mode="key-attribute"/>
                <xsl:text>:</xsl:text>
                <xsl:apply-templates select="." mode="#current"/>
            </xsl:for-each>
            <xsl:text>}</xsl:text>
        </xsl:value-of>
    </xsl:template>
    
    <!-- Template rule for fn:array elements, representing JSON arrays -->
    <xsl:template match="fn:array" mode="indent">
        <xsl:value-of>
            <xsl:variable name="depth" select="count(ancestor::*) + 1"/>
            <xsl:text>[</xsl:text>
            <xsl:for-each select="*">
                <xsl:if test="position() gt 1">
                    <xsl:text>, </xsl:text>
                    <xsl:value-of select="j:indent($depth)"/>
                </xsl:if>
                <xsl:apply-templates select="." mode="#current"/>
            </xsl:for-each>
            <xsl:text>]</xsl:text>
        </xsl:value-of>
    </xsl:template>
    
    <xsl:template match="fn:array" mode="no-indent">
        <xsl:value-of>
            <xsl:text>[</xsl:text>
            <xsl:for-each select="*">
                <xsl:if test="position() gt 1">
                    <xsl:text>,</xsl:text>
                </xsl:if>
                <xsl:apply-templates select="." mode="#current"/>
            </xsl:for-each>
            <xsl:text>]</xsl:text>
        </xsl:value-of>
    </xsl:template>
    
    <!-- Template rule for fn:string elements in which special characters are already escaped -->
    <xsl:template match="fn:string[@escaped='true']" mode="indent no-indent">
        <xsl:sequence select="concat($quot, ., $quot)"/>
    </xsl:template>
    
    <!-- Template rule for fn:string elements in which special characters need to be escaped -->
    <xsl:template match="fn:string[not(@escaped='true')]" mode="indent no-indent">
        <xsl:sequence select="concat($quot, j:escape(.), $quot)"/>
    </xsl:template>
    
    <!-- Template rule for fn:boolean elements -->
    <xsl:template match="fn:boolean" mode="indent no-indent">
        <xsl:sequence select="xs:string(xs:boolean(.))"/>
    </xsl:template>
    
    <!-- Template rule for fn:number elements -->
    <xsl:template match="fn:number" mode="indent no-indent">
        <xsl:value-of select="xs:string(xs:double(.))"/>
    </xsl:template>
    
    <!-- Template rule for JSON null elements -->
    <xsl:template match="fn:null" mode="indent no-indent">
        <xsl:text>null</xsl:text>
    </xsl:template>
    
    <!-- Template rule matching a key within a map where special characters in the key are already escaped -->
    <xsl:template match="fn:*[@key-escaped='true']/@key" mode="key-attribute">
        <xsl:value-of select="concat($quot, ., $quot)"/>
    </xsl:template>
    
    <!-- Template rule matching a key within a map where special characters in the key need to be escaped -->
    <xsl:template match="fn:*[not(@key-escaped='true')]/@key" mode="key-attribute">
        <xsl:value-of select="concat($quot, j:escape(.), $quot)"/>
    </xsl:template>
    
    <!-- Template matching "invalid" elements -->
    <xsl:template match="*" mode="indent no-indent">
        <xsl:param name="fallback" as="(function(element()) as xs:string)?"
            tunnel="yes" required="yes"/>
        <xsl:choose>
            <xsl:when test="exists($fallback)">
                <xsl:value-of select="$fallback(snapshot(.))"/>
            </xsl:when>
            <xsl:otherwise>
                <xsl:message terminate="yes">>Inc</xsl:message>
            </xsl:otherwise>
        </xsl:choose>
    </xsl:template>
    
    <!-- Template rule matching (and discarding) whitespace text nodes in the XML -->
    <xsl:template match="text()[not(normalize-space())]" mode="indent no-indent"/>
    
    <!-- Function to escape special characters -->
    <xsl:function name="j:escape" as="xs:string" visibility="final">
        <xsl:param name="in" as="xs:string"/>
        <xsl:value-of>
            <xsl:for-each select="string-to-codepoints($in)">
                <xsl:choose>
                    <xsl:when test=". gt 65535">
                        <xsl:value-of select="concat('\u', j:hex4((. - 65536) idiv 1024 + 55296))"/>
                        <xsl:value-of select="concat('\u', j:hex4((. - 65536) mod 1024 + 56320))"/>
                    </xsl:when>
                    <xsl:when test=". = 34">\"</xsl:when>
                    <xsl:when test=". = 92">\\</xsl:when>
                    <xsl:when test=". = 08">\b</xsl:when>
                    <xsl:when test=". = 09">\t</xsl:when>
                    <xsl:when test=". = 10">\n</xsl:when>
                    <xsl:when test=". = 12">\f</xsl:when>
                    <xsl:when test=". = 13">\r</xsl:when>
                    <xsl:when test=". lt 32 or (. ge 127 and . le 160)">
                        <xsl:value-of select="concat('\u', j:hex4(.))"/>
                    </xsl:when>
                    <xsl:otherwise>
                        <xsl:value-of select="codepoints-to-string(.)"/>
                    </xsl:otherwise>
                </xsl:choose>
            </xsl:for-each>
        </xsl:value-of>
    </xsl:function>
    
    <!-- Function to convert a UTF16 codepoint into a string of four hex digits -->
    <xsl:function name="j:hex4" as="xs:string" visibility="final">
        <xsl:param name="ch" as="xs:integer"/>
        <xsl:variable name="hex" select="'0123456789abcdef'"/>
        <xsl:value-of>
            <xsl:value-of select="substring($hex, $ch idiv 4096 + 1, 1)"/>
            <xsl:value-of select="substring($hex, $ch idiv 256 mod 16 + 1, 1)"/>
            <xsl:value-of select="substring($hex, $ch idiv 16 mod 16 + 1, 1)"/>
            <xsl:value-of select="substring($hex, $ch mod 16 + 1, 1)"/>
        </xsl:value-of>
    </xsl:function>
    
    <!-- Function to output whitespace indentation based on the depth of the node supplied as a parameter -->
    
    <xsl:function name="j:indent" as="text()" visibility="public">
        <xsl:param name="depth" as="xs:integer"/>
        <xsl:value-of select="'&#xa;', string-join((1 to ($depth + 1) * $indent-spaces) ! ' ', '')"/>
    </xsl:function>
    
</xsl:package>
