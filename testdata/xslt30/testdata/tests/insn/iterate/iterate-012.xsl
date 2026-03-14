<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">

<!-- Test of xsl:iterate; error; two params have the same name -->

<xsl:output indent="yes"/>

<xsl:template match="/">
<out>
  <xsl:iterate select="//ITEM">
    <xsl:param name="basketCost" as="xs:decimal" select="0"/>
    <xsl:param name="basketCost" as="xs:decimal" select="0"/>
    <xsl:choose>
      <xsl:when test="$basketCost gt 12.00">
        <xsl:break/>
      </xsl:when>
      <xsl:otherwise>
        <item cost="{$basketCost}"><xsl:copy-of select="TITLE"/></item>
        <xsl:next-iteration>
          <xsl:with-param name="basketCost" select="$basketCost + (xs:decimal(PRICE), 0)[1]"/>
        </xsl:next-iteration>
      </xsl:otherwise>
    </xsl:choose>  
  </xsl:iterate>
</out>
</xsl:template>

</xsl:stylesheet>