<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
       
  
  <!-- within a streaming template, capture for-each output in an attribute -->
   
  <xsl:mode name="s" streamable="yes"/>
       
  <xsl:output method="xml" indent="yes" encoding="UTF-8" />
  
  <xsl:param name="sep" select="'|'" as="xs:string"/>
   
  <xsl:template name="main" match="/">
    <out>
      <xsl:source-document streamable="true" href="loans.xml"><xsl:apply-templates select="." mode="s"/></xsl:source-document>
    </out>
  </xsl:template>
  
  <xsl:template match="myroot" mode="s">
    <chap>
      <xsl:for-each select="outermost(.//Loan)">
        <loan>
          <xsl:attribute name="statuses">
            <xsl:for-each select="LoanStatus/LoanStatusType">
               <xsl:if test="position() ne 1">|</xsl:if>
               <xsl:value-of select="upper-case(.)"/>               
            </xsl:for-each>
          </xsl:attribute>  
        </loan>
      </xsl:for-each>
    </chap>
  </xsl:template>
       
</xsl:transform>

