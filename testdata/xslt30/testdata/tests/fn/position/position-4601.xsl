<?xml version="1.0" encoding="UTF-8"?>
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: position107 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 4.1 -->
    <!-- Purpose: Test for last() on 'preceding-sibling::' axis. -->

<xsl:output method="xml" encoding="UTF-8" indent="no"/>

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="/doc/north"/>
  </out>
</xsl:template>

<xsl:template match="north">
  <xsl:text>last preceding: </xsl:text>
  <xsl:value-of select="name(preceding-sibling::*[last()])"/>
</xsl:template>

</xsl:stylesheet>