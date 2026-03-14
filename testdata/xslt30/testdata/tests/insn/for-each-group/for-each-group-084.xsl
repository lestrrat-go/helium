<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xsd="http://www.w3.org/2001/XMLSchema"
  xmlns:k="urn:X-Ken"
  exclude-result-prefixes="xsd k"
  version="3.0">

<k:test>
  <_Equationdefinitiontext type="p">a <__TAB/>is an estimate of the border processing costs for the levy period, as adjusted under subclause (2)</_Equationdefinitiontext>
</k:test>

<xsl:template match="/">
  <result>
    <local>
      <xsl:apply-templates select="document('')/*/k:test/*"/>
    </local>
  </result>
</xsl:template>
  
<xsl:template match="_Equationdefinitiontext">
  <variable-def>
    <xsl:for-each-group select="node()" 
                    group-starting-with="__TAB[not(preceding-sibling::__TAB)]">
      <xsl:comment select="'DEBUG',last(),position(),current-group()!string(.)"/>
      <xsl:choose>
        <xsl:when test="last()=1">
          <!--then there is no variable marked up-->
          <variable/>
          <para><text>
            <xsl:apply-templates select="current-group()"/>
          </text></para>
        </xsl:when>
        <xsl:when test="position()=1">
          <!--then this must be the variable-->
          <variable><xsl:apply-templates select="current-group()"/></variable>
        </xsl:when>
        <xsl:otherwise>
          <!--then this must be the description starting with TAB-->
          <para><text>
            <xsl:apply-templates select="current-group()[position()>1]"/>
          </text></para>
        </xsl:otherwise>
      </xsl:choose>
    </xsl:for-each-group>
  </variable-def>
</xsl:template>

</xsl:stylesheet>