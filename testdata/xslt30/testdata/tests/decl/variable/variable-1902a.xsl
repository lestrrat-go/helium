<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: to be imported  -->

<xsl:param name="test" select="'test value set in var27side'"/>

<xsl:template match="b">
  <side><xsl:value-of select="$test"/></side>
  <xsl:apply-templates/>
</xsl:template>

</xsl:stylesheet>