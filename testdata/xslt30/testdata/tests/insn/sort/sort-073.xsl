<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Provide sequence-constructor content for an xsl:perform-sort instruction -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:template match="/">
	     <out>
         <xslt:perform-sort>
		          <xslt:sort select="xs:string(.)" data-type="text" order="descending"/>
		          <xslt:for-each select="doc/a">
			            <xslt:value-of select="."/>
		          </xslt:for-each>
		       </xslt:perform-sort>
      </out>
   </xslt:template>

   <xslt:template match="text()"/>
</xslt:transform>
