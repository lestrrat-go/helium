<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#apply-imports?>
    <!-- Purpose: See what happens to apply-imports when there is a param stack in place. -->

<xsl:import href="fragments/impwparam.xsl"/>

<xsl:output method="xml" encoding="UTF-8" indent="no"/>

<xsl:template match="doc">
  <out>
    <xsl:apply-templates select="*">
      <xsl:with-param name="p1" select="'top'"/>
    </xsl:apply-templates>
  </out>
</xsl:template>

<xsl:template match="tag">
  <xsl:param name="p1" select="'fallback'"/>
  <main-t><xsl:value-of select="$p1"/></main-t>
  <div>
    <xsl:apply-imports/>
  </div>
</xsl:template>

<!-- No template for "bag" here, so it will use the one in the imported file. -->

</xsl:stylesheet>