<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  exclude-result-prefixes="#all" version="3.0">
  
  <xsl:param name="STREAMABLE" static="yes" select="false()"/>
  
  <xsl:param name="c" select="'country'"/>
  
  <xsl:mode _streamable="{$STREAMABLE}"/>
  
  <xsl:template match="/">
    <xsl:variable name="v" as="map(xs:string, item()*)*">
      <xsl:apply-templates/>
    </xsl:variable>
    <out>
      <xsl:for-each select="$v">
        <country name="{?country}" pop="{?sum-pop}" cities="{?cities}"/>     
      </xsl:for-each>
    </out>
  </xsl:template>
  
  <xsl:template match="cities">
      <xsl:fork>
        <xsl:for-each-group select="city" group-by="@country">
          <xsl:sequence select="map{'country':current-grouping-key(),
            'sum-pop': sum(current-group()/@pop),
            'cities': current-group()/@name/string(),
            $c : 'uk'}"/>
        </xsl:for-each-group>
      </xsl:fork>    
  </xsl:template>
  
</xsl:stylesheet>