<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: to be imported  -->

<xsl:param name="test" select="'set in var29side, should have highest precedence'"/>

<xsl:template match="a">
  <side><xsl:value-of select="$test"/></side>
  <xsl:apply-templates/>
</xsl:template>

<xsl:template match="c">
  <side-again><xsl:value-of select="$test"/></side-again>
</xsl:template>

</xsl:stylesheet>