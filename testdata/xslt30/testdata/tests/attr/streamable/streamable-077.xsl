<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
       
  
    <!-- within a streaming template, for-each instruction whose select expression depends on the context but which
         doesn't select streamed nodes -->
     
    <xsl:mode name="s" streamable="yes"/>
         
    <xsl:output method="xml" indent="no" encoding="UTF-8" />
    
    <xsl:param name="sep" select="'|'" as="xs:string"/>
     
    <xsl:template name="main" match="/">
      <out>
        <xsl:source-document streamable="true" href="loans.xml"><xsl:apply-templates select="." mode="s"/></xsl:source-document>
      </out>
    </xsl:template>
    
    <xsl:template match="Loan" mode="s">
      <loan>
        <xsl:for-each select="1 to count(.//*)">
          <xsl:value-of select="concat(., ' ')"/>
        </xsl:for-each>
      </loan>
    </xsl:template>
    
    <xsl:template match="text()" mode="#all"/>
       
</xsl:transform>

