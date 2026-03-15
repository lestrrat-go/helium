<?xml version="1.0" encoding="UTF-8"?>
<xsl:package 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  xmlns:xd="http://www.oxygenxml.com/ns/doc/xsl"
  xmlns:oupdtg="http://www.160.com/oupdtg" 
  name="http://www.160.com/runner"
  package-version="0.1"
  exclude-result-prefixes="#all" 
  version="3.0">
  
  
  <xd:doc scope="stylesheet">
    <xd:desc>
      <xd:p><xd:b>Created on:</xd:b> 2017-07-04</xd:p>
      <xd:p><xd:b>Author:</xd:b> Mark Dunn</xd:p>
      <xd:p><xd:b>Description:</xd:b> Test inclusion of packages. https://saxonica.plan.io/issues/3333</xd:p>
    </xd:desc>
  </xd:doc>


  <xsl:use-package name="http://www.160.com/logger" package-version="0.1">
    <xsl:accept component="*" names="*" visibility="private"/>    
  </xsl:use-package>
  <xsl:use-package name="http://www.160.com/generalFunctions" package-version="0.1">
    <xsl:accept component="*" names="*" visibility="private"/>
  </xsl:use-package>
  
  <xsl:mode name="transform" visibility="public"/>
  
  

  <xsl:template match="*" mode="transform">
    <output>
      <xsl:call-template name="oupdtg:log"/>
      <xsl:copy-of select="oupdtg:getCity()"/>
    </output>
  </xsl:template>
  
</xsl:package>
