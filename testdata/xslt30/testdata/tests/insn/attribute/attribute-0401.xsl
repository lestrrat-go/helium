<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet version="2.0"
xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

   <?spec xslt#creating-attributes?>
<!-- This test is designed to exercise the different paths in the
     Saxon implementation of xsl:attribute -->

<xsl:param name="p" select="'expression'"/>

<xsl:template match="/">
<html>
<xsl:variable name="x">c<xsl:message>hi</xsl:message></xsl:variable>
<xsl:attribute name="one">simple</xsl:attribute>
<xsl:attribute name="two"><xsl:value-of select="$p"/></xsl:attribute>
<xsl:attribute name="three"><xsl:if test="$p">complex</xsl:if></xsl:attribute>
<xsl:attribute name="four"><xsl:value-of select="."/></xsl:attribute>
</html>
</xsl:template>

</xsl:stylesheet>