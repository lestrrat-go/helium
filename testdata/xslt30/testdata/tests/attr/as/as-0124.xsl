<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test with a local xsl:param, where @select is an empty sequence obtained at run time and @as=xs:double*. 
  				Verify the parameter is of type xs:double*.  -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:template match="/doc">
      <xslt:param name="par" select="/doc/item" as="xs:double*"/>
      <out>
         <xslt:value-of select="$par instance of xs:double*"/>
         <xslt:value-of select="$par instance of xs:double"/>
      </out>
   </xslt:template>
</xslt:transform>
