<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" exclude-result-prefixes="xs math"
    version="3.0">

    <xsl:param name="STREAMABLE" static="yes" as="xs:boolean" select="true()"/>

    <xsl:param name="chunk-size" as="xs:integer" select="10"/>

    <xsl:mode _streamable="{$STREAMABLE}" on-no-match="shallow-copy"/>

    <xsl:output indent="yes"/>

    <xsl:param name="tree-to-insert">
        <root>
            <data-list/>
        </root>
    </xsl:param>

    <xsl:template match="/">
        <out>
            <xsl:for-each-group select="root/item"
                group-adjacent="(position() - 1) idiv $chunk-size">
                <xsl:apply-templates select="$tree-to-insert/*">
                    <xsl:with-param name="current-chunk" select="copy-of(current-group())"
                        tunnel="yes"/>
                </xsl:apply-templates>
            </xsl:for-each-group>
        </out>
    </xsl:template>

    <xsl:template match="data-list">
        <xsl:param name="current-chunk" tunnel="yes"/>
        <item-chunk>
            <xsl:copy-of select="$current-chunk"/>
        </item-chunk>
    </xsl:template>

</xsl:stylesheet>
