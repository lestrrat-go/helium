<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
xmlns:xs="http://www.w3.org/2001/XMLSchema" version="3.0"
exclude-result-prefixes="xs">

<!-- Purpose: matching-substring must precede non-matching-substring -->

   <xsl:template name="main">
      <out>        
        <xsl:analyze-string select="'abcdef'" regex='a'>
          <xsl:non-matching-substring>
            <b/>
          </xsl:non-matching-substring>
          <xsl:matching-substring>
            <a/>
          </xsl:matching-substring>
        </xsl:analyze-string>
     </out>
   </xsl:template>
</xsl:transform>
