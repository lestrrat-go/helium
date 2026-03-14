<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: variable3101a -->
  <!-- Purpose: to be imported by variable3101 -->

<xsl:template name="baseTemplate">
  <xsl:param name="baseParam0"/><!-- Not actually used -->
  <xsl:param name="baseParam1"/>
  <xsl:copy-of select="$baseParam1"/>
  <OK/>
</xsl:template>

<xsl:template name="baseSubTemplate">
  <xsl:param name="baseSubParam0"/><!-- Not actually used -->
  <OK-too/>
</xsl:template>

</xsl:stylesheet>
