<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://www.examples.com/my"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                version="2.0">
<!-- Purpose: Test case to show that the @data-type of the xsl:sort instruction can be used to
  				sort data types that are not comparable (eg:xs:QName), as text or number.-->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:variable name="v"
                  select="(xs:QName('my:z'),xs:QName('my:g'),xs:QName('my:a'))"
                  as="xs:QName*"/>

   <xslt:template match="/">
	     <out>
         <xslt:for-each select="$v">
    		      <xslt:sort select="." data-type="text" order="descending"/>
    		      <xslt:text>
</xslt:text>
   			      <txt>
               <xslt:value-of select="."/>
            </txt>
  		     </xslt:for-each>
         <xslt:text>
</xslt:text>
         <xslt:for-each select="$v">
    		      <xslt:sort select="." data-type="number" order="descending"/>
    		      <xslt:text>
</xslt:text>
   			      <num>
               <xslt:value-of select="."/>
            </num>
  		     </xslt:for-each>
         <xslt:text>
</xslt:text>
      </out>
   </xslt:template>

   <xslt:template match="text()"/>
</xslt:transform>
