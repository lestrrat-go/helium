<?xml version="1.0" encoding="UTF-8"?> <xsl:transform version="2.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xi="http://www.w3.org/2001/XInclude">

   <?spec xslt#document?>
   <?spec xslt#unparsed-text?>

<xsl:template match="*|@*">
   <xsl:copy>
     <xsl:apply-templates select="@*|node()"/>
   </xsl:copy>
</xsl:template>

<xsl:template match="xi:include">
    <xsl:variable name="doc" select="@href"/>
    <xsl:apply-templates select="document($doc)"/> </xsl:template>

<xsl:template match="xi:include[@parse='text']" priority="2">
    <xsl:variable name="doc" select="@href"/>
    <xsl:value-of select="unparsed-text($doc, 'UTF-8')"/> </xsl:template>

</xsl:transform>