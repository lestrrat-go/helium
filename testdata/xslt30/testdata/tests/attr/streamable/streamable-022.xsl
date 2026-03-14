<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
       
  
  <!-- get local properties of parent node; including "instance of" -->
   
  <xsl:import-schema schema-location="loans.xsd"/>

  <xsl:mode name="s" streamable="yes"/>
  <xsl:mode name="t" streamable="yes"/>
       
  <xsl:output method="xml" indent="yes" encoding="UTF-8" />
  <xsl:strip-space elements="Pool"/>
  
    
  <xsl:template name="main" match="/">
    <out>
      <xsl:source-document streamable="true" href="loans.xml" validation="strict">
        <xsl:apply-templates select="." mode="s"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <xsl:template match="*" mode="s">
    <xsl:copy>
      <xsl:copy-of select="@*"/>
      <xsl:apply-templates mode="s"/>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="Loan" mode="s">
    <parent name="{name(..)}" local-name="{local-name(..)}" namespace-uri="{namespace-uri(..)}"
            node-name="{node-name(..)}" base-uri="{substring-after(base-uri(..), 'streamable/')}" 
            generate-id="{generate-id(..)}" nilled="{nilled(..)}"
            boolean="{boolean(..)}" exists="{exists(parent::Pool)}" empty="{empty(parent::Loop)}" not="{not(..)}" 
            instance-of-element="{.. instance of element(Pool)}"/>
    <xsl:variable name="copy">
      <xsl:copy-of select="child::node()"/>
    </xsl:variable>
    <children count="{count($copy/*)}"/>
  </xsl:template>
    
</xsl:transform>

