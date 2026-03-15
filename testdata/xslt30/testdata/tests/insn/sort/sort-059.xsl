<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Sort values that are cast to dates dynamically in the select attribute -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:variable name="x">
	     <a xmlns:xsl="http://www.w3.org/1999/XSL/Transform">1999-12-31</a>
	     <a xmlns:xsl="http://www.w3.org/1999/XSL/Transform">2000-12-31</a>
	     <a xmlns:xsl="http://www.w3.org/1999/XSL/Transform">1999-11-01</a>
	     <a xmlns:xsl="http://www.w3.org/1999/XSL/Transform">1990-10-30</a>
	     <a xmlns:xsl="http://www.w3.org/1999/XSL/Transform">2005-11-03</a>
	     <a xmlns:xsl="http://www.w3.org/1999/XSL/Transform">2006-04-04</a>
  </xslt:variable>

   <xslt:template match="/">
      <out>
         <xslt:text>
</xslt:text>
         <xslt:for-each select="$x/a">
            <xslt:sort select="xs:date(.)" order="descending"/>
            <Date>
               <xslt:value-of select="."/>
            </Date>
            <xslt:text>
</xslt:text>
        </xslt:for-each>
      </out>                                       
  </xslt:template>

   <xslt:template match="text()"/>
</xslt:transform>
