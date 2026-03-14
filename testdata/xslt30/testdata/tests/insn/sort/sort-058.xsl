<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: String sort on a set that includes empty strings. -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:variable name="var"
                  select="('a','b','',' ','g','x','')"
                  as="xs:anyAtomicType*"/>

   <xslt:template match="/">
	     <out>
         <xslt:for-each select="$var">
    		      <xslt:sort select="." data-type="text" order="ascending"/>
   			      <xslt:value-of select="."/>
   			      <xslt:value-of select="' | '"/>
  		     </xslt:for-each>
      </out>
   </xslt:template>
</xslt:transform>
