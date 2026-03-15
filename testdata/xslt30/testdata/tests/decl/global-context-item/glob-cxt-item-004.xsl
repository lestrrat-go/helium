<xsl:package
  name="http://www.w3.org/xslt30-test/glob-cxt-item-001"
  package-version="1.0"
  version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">
  
    <xsl:mode name="m" visibility="public"/>


    <xsl:global-context-item use="optional"/>
    
    <xsl:variable name="g" select="/doc"/>

    <xsl:template match="/" mode="m">
      <out>
         <xsl:copy-of select="$g"/>
      </out>
    </xsl:template> 
  
    <xsl:global-context-item use="optional"/>
          
  </xsl:package>