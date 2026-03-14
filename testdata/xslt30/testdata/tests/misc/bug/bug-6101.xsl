<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform version="2.0"
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
   
<?spec fo#func-concat?>
<!-- causes a stack overflow in 7.5.1. No idea why; it works under 7.6 -->
   
<xsl:template match="/">
  <xsl:apply-templates>
     <xsl:with-param name="prefix" select="xxx"/>
  </xsl:apply-templates>
</xsl:template>

<xsl:template match="a">
    <xsl:param name="prefix"/>
    <xsl:param name="scope"/>
  <x>
    <xsl:apply-templates select="b">
        <xsl:with-param name="prefix2" select="concat($prefix, @name, '.')"/>
    </xsl:apply-templates>
  </x>
</xsl:template>

<xsl:template match="b">
  <xsl:param name="prefix2"/>
  [<xsl:value-of select="$prefix2"/>]
</xsl:template>

</xsl:transform>