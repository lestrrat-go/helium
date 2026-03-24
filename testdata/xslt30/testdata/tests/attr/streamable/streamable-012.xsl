<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    version="3.0">
       
  
  <!-- Test apply-templates to attributes, namespaces, comments, and PIs in streaming mode -->
   
  <xsl:import-schema schema-location="loans.xsd"/>

  <xsl:mode name="s" streamable="yes"/>
  <xsl:mode name="t" streamable="yes"/>
       
  <xsl:output method="xml" encoding="UTF-8" />
  
  <xsl:param name="use-prefix" select="true()"/>
    
    
  <xsl:template name="main" match="/">
    <out>
      <xsl:source-document streamable="true" href="loans.xml">
        <xsl:apply-templates select="." mode="s"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <xsl:template match="text()" mode="s"/>
  
      
   <xsl:template match="your:extra" mode="s" xmlns:your="http://your.com/ns">
     <extra>
       <xsl:apply-templates select="copy-of(@*)"/>
       <xsl:apply-templates mode="t"/>
     </extra>
   </xsl:template>
   
   <xsl:template match="@*">
     <attribute name="{name()}" select="{.}">
       <xsl:if test="namespace-uri() != ''">
          <xsl:namespace name="{substring-before(name(), ':')}" select="namespace-uri()"/>
       </xsl:if>
     </attribute>
   </xsl:template>
   
   <xsl:template match="text()" mode="t">
     <text><xsl:copy-of select="."/></text>
   </xsl:template>
   
   <xsl:template match="comment()" mode="t">
     <comment><xsl:copy-of select="."/></comment>
   </xsl:template>
   
   <xsl:template match="processing-instruction()" mode="t">
     <pi><xsl:copy-of select="."/></pi>
   </xsl:template>
   

</xsl:transform>

