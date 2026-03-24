<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:t="http://ns.example.com/val12/"
>

<!-- PURPOSE: effect of xsl:copy with validation on the nilled property -->
<?spec fo#func-nilled?>

  <xsl:import-schema namespace="http://ns.example.com/val12/" schema-location="validation-12.xsd"/>

  <xsl:template match="/">
  <z>
    <xsl:variable name="s" as="element()">
      <xsl:copy select="//t:test" validation="strict">
        <xsl:copy-of select="@*"/>
      </xsl:copy>
    </xsl:variable>
    <xsl:variable name="l" as="element()">
      <xsl:copy select="//t:test" validation="lax">
        <xsl:copy-of select="@*"/>
      </xsl:copy>
    </xsl:variable>
    <xsl:variable name="t" as="element()">
      <xsl:copy select="//t:test" validation="preserve">
        <xsl:copy-of select="@*"/>
      </xsl:copy>
    </xsl:variable>
    <xsl:variable name="u" as="element()">
      <xsl:copy select="//t:test" validation="strip">
        <xsl:copy-of select="@*"/>
      </xsl:copy>
    </xsl:variable>
    <xsl:apply-templates select="($s, $l, $t, $u)"/>
  </z>
  </xsl:template>
  
  <xsl:template match="t:test">
    <a><xsl:value-of select="nilled(.)" /></a>
  </xsl:template>

</xsl:stylesheet>
