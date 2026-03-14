<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test with an xs:anyURI value as a child of LRE in the sequence constructor of a global xsl:param and @as=xs:anyURI+.
  				Verify the parameter is of type xs:anyURI+. -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:param name="par" as="xs:anyURI+">
	     <elem>
         <xslt:value-of select="xs:anyURI('test.org')"/>
      </elem>
   </xslt:param>

   <xslt:template match="/doc">
      <out>
         <xslt:value-of select="$par instance of xs:anyURI+"/>
         <xslt:value-of select="$par instance of xs:anyURI"/>
      </out>
   </xslt:template>
</xslt:transform>
