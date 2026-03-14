<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
  
  <!-- streaming, xsl:choose as child of xsl:source-document; an xsl:choose that Saxon optimizes -->
  
  <xsl:strip-space elements="*"/>
  
    
  <xsl:template name="main">
    <out>
      <xsl:source-document streamable="yes" href="../docs/big-transaction-content.xml">
        <xsl:for-each select="account/transaction">
          <xsl:choose>
            <xsl:when test="string(@date) = '2007-01-01'">
              <D1><xsl:value-of select="value/@amount"/></D1>
            </xsl:when>
            <xsl:when test="string(@date) = '2007-01-02'">
              <D2><xsl:value-of select="value/@amount"/></D2>
            </xsl:when>
            <xsl:when test="string(@date) = '2007-01-03'">
              <D3><xsl:value-of select="value/@amount"/></D3>
            </xsl:when>
            <xsl:when test="string(@date) = '2007-01-04'">
              <D4><xsl:value-of select="value/@amount"/></D4>
            </xsl:when>
            <xsl:when test="string(@date) = '2007-01-05'">
              <D5><xsl:value-of select="value/@amount"/></D5>
            </xsl:when>
            <xsl:when test="string(@date) = '2007-01-06'">
              <D6><xsl:value-of select="value/@amount"/></D6>
            </xsl:when>
            <xsl:when test="string(@date) = '2007-01-07'">
              <D7><xsl:value-of select="value/@amount"/></D7>
            </xsl:when>
          </xsl:choose>
        </xsl:for-each>
      </xsl:source-document>
    </out>  
  </xsl:template> 

       
</xsl:transform>

