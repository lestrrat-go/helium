<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
<!-- Purpose: test push-mode validation (in Saxon) against a document node test.-->
  
  <xsl:variable name="data">
    <one>
      <two>
        <three>3</three>
      </two>
    </one>
  </xsl:variable>

   <xsl:template match="/">
      <out>
        <xsl:call-template name="content"/>
      </out>
   </xsl:template>
  
   <xsl:template name="content" as="document-node(element(four))">
     <xsl:document>
       <xsl:apply-templates select="$data/*"/>
     </xsl:document>
   </xsl:template>
</xsl:transform>
