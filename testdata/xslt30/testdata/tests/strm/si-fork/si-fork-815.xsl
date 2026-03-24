<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
  xmlns:saxon="http://saxon.sf.net/" 
  xmlns:mf="http://example.com/mf" 
  exclude-result-prefixes="#all" 
  version="3.0">
  
  <xsl:mode streamable="yes"/>
  
  <xsl:output method="json" indent="no"/>
  
  <xsl:function name="mf:group" as="array(xs:decimal)*" streamability="absorbing">
    <xsl:param name="input" as="element()"/>
    <xsl:fork>
      <xsl:for-each-group select="$input/transaction" group-by="@date">
        <xsl:sequence select="array { current-group()!@value!xs:decimal(.) }"/>
      </xsl:for-each-group>
    </xsl:fork>        
  </xsl:function>    
  
  <xsl:template match="account">
    <xsl:sequence select="array { mf:group(.) }"/>
  </xsl:template>
  
  <xsl:template match="/">
    <xsl:apply-templates select="account"/>
  </xsl:template>
  
</xsl:stylesheet>