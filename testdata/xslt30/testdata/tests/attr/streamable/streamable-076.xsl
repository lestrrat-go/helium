<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes=" xs"
  version="3.0">
       
  
    <!-- within a streaming template, for-each instruction whose body doesn't drill down -->
     
    <xsl:mode name="s" streamable="yes"/>
         
    <xsl:output method="xml" indent="no" encoding="UTF-8" />
    
    <xsl:param name="sep" select="'|'" as="xs:string"/>
     
    <xsl:template name="main" match="/">
      <out>
        <xsl:source-document streamable="true" href="loans.xml"><xsl:apply-templates select="." mode="s"/></xsl:source-document>
      </out>
    </xsl:template>
    
    <xsl:template match="myroot" mode="s">
      <chap>
        <xsl:for-each select=".//Loan">
          <loan atts="{count(@*)}"/>
        </xsl:for-each>
      </chap>
    </xsl:template>
       
</xsl:transform>

