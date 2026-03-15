<?xml version="1.0" encoding="UTF-8"?>
<xsl:package 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  xmlns:xd="http://www.oxygenxml.com/ns/doc/xsl"
  xmlns:oupdtg="http://www.160.com/oupdtg" 
  name="http://www.160.com/logger"
  package-version="0.1"
  exclude-result-prefixes="#all" 
  version="3.0">
  
 

  <xsl:use-package name="http://www.160.com/generalVariables" package-version="0.1">
    <xsl:accept component="*" names="*" visibility="private"/>
  </xsl:use-package>
  

  <xsl:template name="oupdtg:log" visibility="public">
    <log>
      <xsl:value-of select="$oupdtg:city"/>
    </log>
  </xsl:template>
  
</xsl:package>
