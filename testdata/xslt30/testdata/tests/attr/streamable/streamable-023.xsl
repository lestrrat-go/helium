<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
       
  
  <!-- get local properties of attribute of ancestor node; including "instance of" -->
   
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
    <version value="{ancestor::ACERequest/@version}" 
            name="{name(ancestor::ACERequest/@version)}" 
            local-name="{local-name(ancestor::ACERequest/@version)}" 
            namespace-uri="{namespace-uri(ancestor::ACERequest/@version)}"
            node-name="{node-name(ancestor::ACERequest/@version)}" 
            base-uri="{substring-after(base-uri(ancestor::ACERequest/@version), 'streamable/')}" 
            generate-id="{string-length(generate-id(ancestor::ACERequest/@version)) lt 10000}" 
            nilled="{nilled(ancestor::ACERequest/@version)}"
            boolean="{boolean(ancestor::ACERequest/@version)}" 
            exists="{exists(ancestor::ACERequest/@VERSION)}" 
            empty="{empty(ancestor::ACERequest/@version)}" 
            not="{not(ancestor::ACERequest/@version)}" 
            instance-of-att="{ancestor::ACERequest/@* instance of attribute(VERSION)}"/>
    <xsl:variable name="copy">
      <xsl:copy-of select="child::node()"/>
    </xsl:variable>
    <children count="{count($copy/*)}"/>
  </xsl:template>
    
</xsl:transform>

