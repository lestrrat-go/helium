<xsl:transform
  version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

    <xsl:global-context-item use="absent" as="document-node(element(doc))"/>
    
    <xsl:variable name="g" select="/doc"/>

    <xsl:template name="xsl:initial-template">
      <out>
         <xsl:copy-of select="$g"/>
      </out>
    </xsl:template> 
  
  </xsl:transform>