<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes=" xs" version="3.0">

  <!-- streaming, xsl:choose inside for-each, with motionless conditions -->

  <xsl:strip-space elements="*"/>


  <xsl:template name="main">
    <out>
      <xsl:source-document streamable="yes" href="../docs/bullets.xml">
        <xsl:for-each select="/*/*">
          <xsl:choose>
            <xsl:when test="name() = 'p'"><para/></xsl:when>
            <xsl:when test="@noWay"><sorcery/></xsl:when>
            <xsl:when test="ancestor::*/@noWay"><blackMagic/></xsl:when>
            <xsl:when test="generate-id() = 'improbably95281'"><justBadLuck/></xsl:when>
            <xsl:when test="self::important"><egotistic/></xsl:when>
            <xsl:otherwise><xsl:copy-of select="."/></xsl:otherwise>
          </xsl:choose>
        </xsl:for-each>  
      </xsl:source-document>
    </out>
  </xsl:template>

</xsl:transform>
