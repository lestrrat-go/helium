<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map" xmlns:mf="http://example.com/mf"
    exclude-result-prefixes="xs map mf" version="3.0">

    <xsl:function name="mf:f1">
        <xsl:param name="k1"/>
        <xsl:param name="k2"/>
        <xsl:try>
            <xsl:map>
                <xsl:map-entry key="$k1" select="true()"/>
                <xsl:map-entry key="$k2" select="false()"/>
            </xsl:map>
            <xsl:catch errors="*:XTDE3365"/> 
        </xsl:try>
    </xsl:function>

    <xsl:template match="/" name="xsl:initial-template">
        <out>
            <a><xsl:value-of select="mf:f1(3,3) instance of map(*)"/></a>
            <b><xsl:value-of select="mf:f1(xs:integer(3), xs:double(3.0e0)) instance of map(*)"/></b>
            <c><xsl:value-of select="mf:f1('http://w3.org', xs:anyURI('http://w3.org')) instance of map(*)"/></c>
            <d><xsl:value-of select="mf:f1(xs:double('NaN'), xs:float('NaN')) instance of map(*)"/></d>
            <e><xsl:value-of select="mf:f1(xs:time('18:15:00-05:00'), xs:time('23:15:00Z')) instance of map(*)"/></e>
        </out>
    </xsl:template>

</xsl:stylesheet>
