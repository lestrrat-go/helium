<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">

<!-- Test of xsl:iterate; uses a parameter. Error, xsl:break must come last -->

<xsl:output indent="yes"/>

<xsl:template match="/">
<out>
  <xsl:iterate select="//ITEM">
    <xsl:param name="basketCost" as="xs:decimal" select="0"/>
    <item cost="{$basketCost}"><xsl:copy-of select="TITLE"/></item>
    <xsl:if test="2 + 2 = 5">
       <xsl:break/>
       <ouch/>
    </xsl:if>
  </xsl:iterate>
</out>
</xsl:template>

</xsl:stylesheet>