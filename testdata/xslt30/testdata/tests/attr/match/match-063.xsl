<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" version="3.0">

    <?spec xslt#patterns?>
    <!-- Purpose: "except" in pattern -->
    <!-- See bug 30110 regarding template rule precedence: changed order of rules -->


    <xsl:template match="/">
        <out>
            <xsl:apply-templates/>
        </out>
    </xsl:template>

    <xsl:template match="foo">
        <special-foo>
            <xsl:apply-templates/>
        </special-foo>
    </xsl:template>

    <xsl:template match="foo except /*/*">
        <foo2>
            <xsl:apply-templates/>
        </foo2>
    </xsl:template>





</xsl:stylesheet>
