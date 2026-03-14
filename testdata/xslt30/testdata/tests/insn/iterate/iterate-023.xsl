<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<!-- xsl:on-completion is in the wrong place -->

<xsl:output indent="yes"/>

<xsl:template match="/">
    <xsl:apply-templates select="//page">
       <xsl:with-param name="max" select="1000"/>
    </xsl:apply-templates>
</xsl:template>

<xsl:template match="page">
<xsl:param name="max" as="xs:integer"/>
<out>
<xsl:iterate select="*">
  <xsl:param name="count" select="0"/>
  <xsl:choose>
    <xsl:when test="self::h3">
       <heading><xsl:copy-of select="child::node()"/></heading>
       <xsl:next-iteration>
          <xsl:with-param name="count" select="$count+1"/>
       </xsl:next-iteration>
    </xsl:when>
    <xsl:otherwise>
       <xsl:copy-of select="."/>
    </xsl:otherwise>
  </xsl:choose>
  <xsl:fallback/>
  <xsl:if test="system-property('xsl:version')='3.0'">
    <xsl:on-completion>
      <count><xsl:value-of select="$count"/></count>
    </xsl:on-completion>
  </xsl:if>  
</xsl:iterate>
</out>
</xsl:template>

<xsl:template match="p">
  <xsl:copy-of select="."/>
</xsl:template>

</xsl:stylesheet>