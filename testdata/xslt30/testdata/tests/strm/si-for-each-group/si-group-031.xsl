<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" version="3.0" exclude-result-prefixes="xs">

    <xsl:param name="block-size" as="xs:integer" select="10"/>

    <xsl:output indent="yes"/>
    <xsl:mode streamable="yes"/>

    <xsl:template match="/*">
        <xsl:variable name="root" as="element()">
            <xsl:copy/>
        </xsl:variable>
        <out>
            <xsl:for-each-group select="product" group-adjacent="(position() - 1) idiv $block-size">
                <xsl:result-document href="product{current-grouping-key()}.xml">
                    <xsl:copy select="$root">
                        <!-- 
                            (AB 20161103)
                            Per bug 29482 the following may be considered a higher-order invalid use of current-group() 
                            However, as the analysis shows, this is not the case, as the construct does not *require* multiple
                            evaluations of the streamed node, because its focus-changing construct, here $root, is bound to a
                            grounded node
                         -->
                        <xsl:copy-of select="current-group()"/>
                    </xsl:copy>
                </xsl:result-document>
            </xsl:for-each-group>
        </out>
    </xsl:template>

</xsl:stylesheet>
