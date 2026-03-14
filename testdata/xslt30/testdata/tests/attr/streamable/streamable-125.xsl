<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
       
  
  <!-- within a streaming template, passing a streamed node as a parameter is not allowed, 
       because we can't tell where the callee will navigate -->
   
  <xsl:mode name="s" streamable="yes"/>
       
  <xsl:output method="xml" indent="yes" encoding="UTF-8" />
  
  <xsl:param name="sep" select="'|'" as="xs:string"/>
   
  <xsl:template name="main" match="/">
    <out>
      <xsl:source-document streamable="true" href="strmode037.xml">
        <xsl:apply-templates select="." mode="s"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <xsl:template match="chap" mode="s">
    <chap>
      <xsl:apply-templates select=".//section" mode="s">
        <xsl:with-param name="x" select="../@code" tunnel="yes"/>
      </xsl:apply-templates>
    </chap>
  </xsl:template>
  
  <xsl:template match="section" mode="s">
    <section>
      <xsl:apply-templates select="page-count" mode="s"/>
    </section>
  </xsl:template>
  
  <xsl:template match="page-count" mode="s">
    <xsl:param name="x" tunnel="yes"/>
    <pages>
      <xsl:value-of select="count($x/ancestor::*)"/>
    </pages>  
  </xsl:template>
      
</xsl:transform>

