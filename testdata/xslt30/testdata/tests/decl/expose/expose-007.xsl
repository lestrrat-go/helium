<?xml version="1.0" encoding="UTF-8"?>
<xsl:package
  name="expose007" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
  xmlns:fo="http://www.w3.org/1999/XSL/Format" 
  exclude-result-prefixes="xs math" 
  version="3.0">
  <xsl:mode default-mode="#unnamed"/>
  
  <xsl:expose component="template" names="*" visibility="public"/>
  
  <xsl:template match="*[contains(@class, ' topic/ph ' )]">
    <fo:inline>
      <xsl:apply-templates/>
    </fo:inline>
  </xsl:template>
  
  <xsl:template match="*[contains(@class,' hi-d/i ')]" priority="2">
    <fo:inline font-style="italic">
      <xsl:apply-templates/>
    </fo:inline>
  </xsl:template>
  
  <xsl:template name="main">
    <out/>
  </xsl:template>
  
</xsl:package>