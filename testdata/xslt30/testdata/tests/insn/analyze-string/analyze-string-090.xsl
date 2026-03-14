<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
xmlns:xs="http://www.w3.org/2001/XMLSchema" version="3.0"
exclude-result-prefixes="xs">

<!-- Purpose: match zero-length substring. Allowed in 3.0 -->

   <xsl:variable name="v1" as="xs:string">
Heading

above is an empty line
   
above is a line of three spaces
	
above is a line containing a tab
   </xsl:variable>

   <xsl:template name="main">
      <out>
        <xsl:analyze-string select="$v1" regex="^[\t ]*$" flags="m" expand-text="yes">
          <xsl:non-matching-substring><a nr="{position()}">{.}</a></xsl:non-matching-substring>
        </xsl:analyze-string>
      </out>
   </xsl:template>
</xsl:transform>
