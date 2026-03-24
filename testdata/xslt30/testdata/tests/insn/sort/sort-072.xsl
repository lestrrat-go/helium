<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Nest xsl:sort in xsl:for-each as sequence-constructor for xsl:perform-sort -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:template match="/">
	     <out>
         <xslt:perform-sort>
		          <xslt:sort data-type="number" order="descending"/>
			         <xslt:for-each select="doc/a">
				           <xslt:sort select="xs:double(.)" data-type="number"/>
				           <xslt:value-of select="concat(., ' | ')"/>
			         </xslt:for-each>
		       </xslt:perform-sort>
      </out>
   </xslt:template>

   <xslt:template match="text()"/>
</xslt:transform>
