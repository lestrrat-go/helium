<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: copy62 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 11.3 -->
    <!-- Purpose: Try to copy an attribute, via copy-of, after child element. -->
  <!-- Discretionary: name="add-attribute-after-children" choice="ignore" -->
  <!-- ExpectedException: Attempt to write an attribute after child element -->

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template match="/">
  <out>
    <child/>
    <xsl:copy-of select="doc/@*"/>
  </out>
</xsl:template>

</xsl:stylesheet>