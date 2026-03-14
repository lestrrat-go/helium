<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet version="2.0"
xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<!-- This test is designed to show that fixed attributes are copied 
     and implied attributes are not -->

   <?spec dm#const-infoset-element?>
<xsl:param name="p" select="'expression'"/>

<xsl:template match="/">
<xsl:comment>abcdef</xsl:comment>
    <xsl:copy-of select="page"/>
</xsl:template>

</xsl:stylesheet>