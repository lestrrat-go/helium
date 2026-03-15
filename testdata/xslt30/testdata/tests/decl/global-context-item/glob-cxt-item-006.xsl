<xsl:package
  name="http://www.w3.org/xslt30-test/glob-cxt-item-006"
  package-version="1.0"
  version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

    <xsl:mode/>

    <xsl:global-context-item use="absent"/>
    
    <xsl:variable name="g" select="23"/>

    <xsl:template name="xsl:initial-template" visibility="public">
      <out>
         <xsl:copy-of select="$g"/>
      </out>
    </xsl:template> 
  
    <xsl:include href="glob-cxt-item-006b.xsl"/>
          
  </xsl:package>