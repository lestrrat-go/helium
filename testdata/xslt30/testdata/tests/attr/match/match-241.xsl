<?xml version="1.0"?>
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    expand-text="true"
    version="3.0">

    <?spec xslt#patterns?>
    <!-- Purpose: numeric value of predicate.  -->
    <!-- Purpose: multiple matches, atomic sequence, position and last functions  -->
    
    <xsl:mode warning-on-multiple-match="no"/>
    
    <xsl:template name="xsl:initial-template">
        <out><xsl:apply-templates select="1, 2, 3, 5, 7, 13, 17, 19" /></out>
    </xsl:template>

    <xsl:template match=".[. instance of xs:integer]">
        <xsl:next-match />*<xsl:next-match />
        <xsl:if test="position() != last()">, </xsl:if>
    </xsl:template>

    <xsl:template match=".[. instance of xs:integer]"> = <xsl:next-match /></xsl:template>

    <xsl:template match=".[. instance of xs:integer]">{.*.}<xsl:next-match /></xsl:template>

    <xsl:template match=".[. instance of xs:integer]"> = <xsl:next-match /></xsl:template>

    <xsl:template match=".[. instance of xs:integer]">2<xsl:next-match /></xsl:template>

    <xsl:template match=".[. instance of xs:integer]">^<xsl:next-match /></xsl:template>

    <xsl:template match=".[. instance of xs:integer]">{.}<xsl:next-match /></xsl:template>

    
</xsl:stylesheet>
