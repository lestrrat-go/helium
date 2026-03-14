<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:g="http://www.w3.org/xsl-tests/grouped-transactions"
    exclude-result-prefixes="g"
    version="3.0">
       
  
    <!-- within a streaming template, compute count over the atomized value -->
     
    <xsl:mode streamable="yes"/>
         
    <xsl:output method="xml" indent="no" encoding="UTF-8" />
    
    <xsl:import-schema namespace="http://www.w3.org/xsl-tests/grouped-transactions" schema-location="grouped-transactions.xsd"/>
  
    <xsl:strip-space elements="*"/>
      
    <xsl:template name="main">
      <out>
        <xsl:source-document streamable="true" href="grouped-transactions.xml" validation="strict">
          <xsl:apply-templates select="."/>
        </xsl:source-document>
      </out>
    </xsl:template>
     
    <xsl:template match="g:account">
      <transactions number="{count(data(g:transaction/@value))}"/>
    </xsl:template>
      
</xsl:transform>

