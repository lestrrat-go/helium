<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of select in xsl:for-each with empty sequence -->

   <xslt:variable name="var"> 
		    <doc xmlns:xsl="http://www.w3.org/1999/XSL/Transform">6<num1>1<num2>2<num3>3</num3>
            </num2>
         </num1>
         <num4>4<num6>at3</num6>
         </num4>
         <num5>5</num5>
      </doc>
	  </xslt:variable>

   <xslt:template match="doc">
		    <out>
         <t1>
            <xslt:for-each select="$var//*:aaa">
			  		        <xslt:value-of select="."/>
					          <xslt:text>|</xslt:text>
		    	     </xslt:for-each>
         </t1>
      </out>
	  </xslt:template>

   <xslt:template match="text()"/>
</xslt:transform>
