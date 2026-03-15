<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  version="3.0">
   
   <xsl:param name="param"/>
   
   <xsl:template match="/">
      <xsl:variable name="functionItem" select="if ($param eq 'map') then (map{'1':'1st',
         '2':'2nd'}) else if ($param eq 'array') then ['first','second'] else 'not a function'"/>
      
      <out>
         <xsl:sequence select="$functionItem(//e/@ref, 'arg2')"/>
      </out>
   </xsl:template>
</xsl:stylesheet>