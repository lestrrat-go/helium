<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    version="3.0">
       
  
    <!-- within a streaming template, compute starts-with on a singleton child node -->
     
    <xsl:mode streamable="yes"/>
         
    <xsl:output method="xml" indent="yes" encoding="UTF-8" />
    
    <xsl:strip-space elements="*"/>
      
    <xsl:template name="main">
      <out>
        <xsl:source-document streamable="true" href="transactions.xml">
          <xsl:apply-templates select="."/>
        </xsl:source-document>
      </out>
    </xsl:template>
     
    <xsl:template match="*">
      <xsl:copy>
        <xsl:copy-of select="@*"/>
        <xsl:apply-templates/>
      </xsl:copy>
    </xsl:template>
    
    <xsl:template match="account">
      <account zero-prefixed="{starts-with(account-number, '0')}"/>
    </xsl:template>
      
</xsl:transform>

