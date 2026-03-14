<?xml version='1.0' encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl='http://www.w3.org/1999/XSL/Transform' version='2.0'
                xmlns:fiscus="http://www.fiscus.de">

  <!-- Purpose: Create attribute with QName whose prefix is known. -->

<xsl:output method="xml"/>

<xsl:template match="/">
  <xsl:apply-templates/>
</xsl:template>

<xsl:template match="*">
  <xsl:copy>
    <xsl:attribute name="fiscus:objectID">
      <xsl:number level="any" count="*"/>
    </xsl:attribute>
    <xsl:apply-templates select="@*|node()|comment()|processing-instruction()"/>
  </xsl:copy>
</xsl:template>

</xsl:stylesheet>
