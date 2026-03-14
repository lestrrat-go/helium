<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

        <!-- PURPOSE: test tunnel parameters -->

        <xsl:template match="/">
                <xsl:variable name="a">
                        <aa/>
                </xsl:variable>
                <xsl:apply-templates select="$a/*">
                        <xsl:with-param name="x" tunnel="yes">abc</xsl:with-param>
                </xsl:apply-templates>
        </xsl:template>
        
        <xsl:template match="aa">
                <xsl:variable name="b">
                        <bb/>
                </xsl:variable>
                <xsl:apply-templates select="$b/*"/>
        </xsl:template>
        
        <xsl:template match="bb">
                <xsl:param name="x" tunnel="yes"/>
                <a>x = <xsl:value-of select="$x"/></a>
        </xsl:template>
        
</xsl:stylesheet>
