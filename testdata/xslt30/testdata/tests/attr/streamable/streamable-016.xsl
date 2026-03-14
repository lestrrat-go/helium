<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
       
  
  <!-- xsl:attribute with a sequence constructor -->
   
  <xsl:import-schema schema-location="loans.xsd"/>

  <xsl:mode name="s" streamable="yes"/>
  <xsl:mode name="t" streamable="yes"/>
       
  <xsl:output method="xml" indent="yes" encoding="UTF-8" />
  
  <xsl:param name="suffix" select="']'"/>
    
    
  <xsl:template name="main" match="/">
    <out>
      <xsl:source-document streamable="true" href="loans.xml" validation="strict">
        <xsl:apply-templates select="." mode="s"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <xsl:template match="text()" mode="s"/>
  
  <xsl:template match="LoanToValueRatio|Property" mode="s"/>
  
  <xsl:template match="Loan" mode="s">
    <Loan><xsl:apply-templates mode="s"/></Loan>
  </xsl:template>
  
      
   <xsl:template match="LoanStatusType" mode="s">
     <xsl:attribute name="{local-name(.)}">[<xsl:sequence select="string(.)"/>]</xsl:attribute>
   </xsl:template>
   
   <xsl:template match="UnpaidPrincipalBalanceAmount" mode="s">
     <xsl:attribute name="{local-name(.)}">
       <xsl:text>[</xsl:text>
       <xsl:value-of select="."/>
       <xsl:value-of select="$suffix"/>
     </xsl:attribute>
   </xsl:template>
   
   

</xsl:transform>

