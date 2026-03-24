<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose:  Test case that uses xsl:value of to create text nodes by casting the value 
  				of variables, the variable is of type integer.  -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:variable name="v" select="'global var'"/>

   <xslt:param name="p" as="xs:integer">10</xslt:param>

   <xslt:template match="doc">
  		  <out>
         <xslt:call-template name="creator"/>
      </out>
	  </xslt:template>

   <xslt:template name="creator">
		    <xslt:value-of select="$v"/>
		    <xslt:value-of select="$p"/>		
	  </xslt:template>
</xslt:transform>
