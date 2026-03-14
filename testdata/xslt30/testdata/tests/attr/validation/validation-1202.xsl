<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:t="http://ns.example.com/val12/"
>

<!-- PURPOSE: effect of validation on the nilled property -->
<?spec fo#func-nilled?>
  
  <xsl:import-schema namespace="http://ns.example.com/val12/" schema-location="validation-12.xsd"/>

  <xsl:template match="/">
  <z>
    <xsl:variable name="s" as="document-node()">
      <xsl:copy-of select="." validation="strict"/>
    </xsl:variable>
    <xsl:variable name="l" as="document-node()">
      <xsl:copy-of select="." validation="lax"/>
    </xsl:variable>
    <xsl:variable name="t" as="document-node()">
      <xsl:copy-of select="." validation="preserve"/>
    </xsl:variable>
    <xsl:variable name="u" as="document-node()">
      <xsl:copy-of select="." validation="strip"/>
    </xsl:variable>
    <xsl:apply-templates select="for $X in ($s, $l, $t, $u) return $X//t:test"/>  <!-- need to retain order... -->
  </z>
  </xsl:template>
  
  <xsl:template match="t:test">
    <a><xsl:value-of select="nilled(.)" /></a>
  </xsl:template>

</xsl:stylesheet>
