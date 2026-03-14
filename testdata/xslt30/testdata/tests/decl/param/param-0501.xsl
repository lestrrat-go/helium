<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test case to verify that variables used in Quantified Expressions can 
  				shadow global variables and parameters declared with the same name.  -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:variable name="price" select="2"/>

   <xslt:param name="quant" select="2" as="xs:double"/>

   <xslt:template match="/">
		    <xslt:param name="quant"
                  select="some $price in doc/xs:double(prices) satisfies ($price &gt; 9999999)"/>
		    <xslt:variable name="price" select="doc/prices"/>
		    <out>
         <xslt:value-of select="if ( some $price in doc/prices satisfies ($price &gt; 99999999)  )      then 1      else 0"/>
         <xslt:value-of select="if ( some $price in doc/prices satisfies ($price &gt; 9999999)  )      then 1      else 0"/>
         <xslt:value-of select="if ( every $price in doc/prices satisfies ($price &gt; 999999999)  )      then 1      else 0"/>
         <xslt:value-of select="if ( $quant  )      then 1      else 0"/>
      </out>
	  </xslt:template>
</xslt:transform>
