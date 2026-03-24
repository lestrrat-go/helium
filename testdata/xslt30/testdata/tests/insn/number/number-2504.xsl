<?xml version="1.0"?>
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  
  <!-- Purpose: Test of grouping attributes. Use a grouping-separator that is not likely to be in the environment's locale settings. -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match="doc">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="note">
  <xsl:number level="any" from="chapter" format="(1) " grouping-size="2" grouping-separator=":" />
  <xsl:apply-templates/>
</xsl:template>

</xsl:stylesheet>
