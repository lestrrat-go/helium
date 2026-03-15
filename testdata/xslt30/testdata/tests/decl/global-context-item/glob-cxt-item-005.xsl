<xsl:package
  name="http://www.w3.org/xslt30-test/glob-cxt-item-005"
  package-version="1.0"
  version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">
  
    <xsl:strip-space elements="*"/>

    <xsl:mode name="m" visibility="public"/>

    <xsl:global-context-item use="absent"/>
    
    <xsl:variable name="g" select="/doc"/>

    <xsl:template match="/" mode="m">
      <out>
         <xsl:copy-of select="$g"/>
      </out>
    </xsl:template> 
  
    <xsl:include href="glob-cxt-item-005b.xsl"/>
          
  </xsl:package>