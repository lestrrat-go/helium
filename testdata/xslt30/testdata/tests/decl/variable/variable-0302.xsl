<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

        <!-- PURPOSE: test tunnel parameters -->

        <xsl:template match="/">
                <xsl:call-template name="aa">
                        <xsl:with-param name="x" tunnel="yes">abc</xsl:with-param>
                </xsl:call-template>
        </xsl:template>

        <xsl:template name="aa">
                <xsl:call-template name="bb"/>
        </xsl:template>

        <xsl:template name="bb">
                <xsl:param name="x" tunnel="yes"/>
                <a>
                        <xsl:value-of select="$x"/>
                </a>
        </xsl:template>

</xsl:stylesheet>
