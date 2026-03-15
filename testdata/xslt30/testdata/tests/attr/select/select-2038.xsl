<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of select in xsl:for-each with current function -->

   <t:variable name="var"> 
		    <doc xmlns:xsl="http://www.w3.org/1999/XSL/Transform">6<num1>1<num2>2<num3>3</num3>
            </num2>
         </num1>
         <num4>4<num6>at3</num6>
         </num4>
         <num5>5</num5>
      </doc>
	  </t:variable>

   <t:template match="doc">
		    <out>
         <t1>
            <t:for-each select="name($var//*[string() = current()/@*])">
			  		        <t:value-of select="."/>
					          <t:text>|</t:text>
		    	     </t:for-each>
         </t1>
      </out>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
