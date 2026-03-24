<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:mf="http://example.com/mf"
    exclude-result-prefixes="#all"
    version="3.0">
    
    <xsl:param name="select-expression" static="yes" as="xs:string" select="'root/data/foo'"/>
    
    <xsl:mode on-no-match="shallow-copy"/>
    
    <xsl:output indent="yes"/>
    
    <xsl:param name="doc1">
        <root>
            <data>
                <foo>
                    <value>foo 1, 1</value>
                </foo>
                <bar>
                    <value>bar 1, 1</value>
                </bar>
                <foo>
                    <value>foo 2, 1</value>
                </foo>
            </data>
        </root>
    </xsl:param>
    
    <xsl:param name="doc2">
        <root>
            <data>
                <foo>
                    <value>foo 1, 2</value>
                </foo>
                <bar>
                    <value>bar 1, 2</value>
                </bar>
                <foo>
                    <value>foo 2, 2</value>
                </foo>
                <bar>
                    <value>bar 2, 2</value>
                </bar>
                <foo>
                    <value>foo 3, 2</value>
                </foo>
            </data>
        </root>
    </xsl:param>
    
    <xsl:template name="go">
      <m>
        <xsl:merge>
            <xsl:merge-source for-each-item="$doc1, $doc2" _select="{$select-expression}">
                <xsl:merge-key select="value"/>
            </xsl:merge-source>
            <xsl:merge-action>
               <xsl:sequence select="mf:construct-doc((1 to position())!(.*.*.), .)"/>
            </xsl:merge-action>
        </xsl:merge>
      </m>    
    </xsl:template>
    
    <xsl:function name="mf:construct-doc" as="element(out)">
        <xsl:param name="pos" as="xs:integer*"/>
        <xsl:param name="first-node" as="node()"/>
        <out pos="{$pos}" name="{local-name($first-node)}" value="{$first-node/value}"/>
    </xsl:function>
    

    
</xsl:stylesheet>