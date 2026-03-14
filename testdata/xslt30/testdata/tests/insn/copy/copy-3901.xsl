<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: copy57 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- AdditionalSpec: http://www.w3.org/1999/11/REC-xslt-19991116-errata/#E27 -->
  <!-- Section: 11.3 Using Values of Variables & Parameters with xsl:copy-of. -->
    <!-- Purpose: Use copy-of to put a node-set in an attribute, where all members are text nodes. -->
  <!-- Invalid nodes (non-text) and their content should be ignored. -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match="/">
  <out>
    <xsl:attribute name="attr1">
      <xsl:copy-of select="docs//d/text()"/>
    </xsl:attribute>
  </out>
</xsl:template>

</xsl:stylesheet>