<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
xmlns:xs="http://www.w3.org/2001/XMLSchema" version="3.0"
exclude-result-prefixes="xs">

<!-- Purpose: xsl:fallback must come last -->

   <xsl:template name="main">
      <out>        
        <xsl:analyze-string select="'abcdef'" regex='a'>
          <xsl:matching-substring>
            <a/>
          </xsl:matching-substring>
          <xsl:fallback/>
          <xsl:non-matching-substring>
            <b/>
          </xsl:non-matching-substring>          
        </xsl:analyze-string>
     </out>
   </xsl:template>
</xsl:transform>
