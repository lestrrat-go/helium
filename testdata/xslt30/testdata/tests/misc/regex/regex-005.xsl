<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test replace() function with a regular expression not known until run-time. -->
  <?spec fo#func-replace?>
  
  <xsl:param name="regex">ra</xsl:param>

  <xsl:template match="/">
    <out>
      <true result="{replace('abracadabra', $regex, 'XX')}"/>
    </out>
  </xsl:template>

</xsl:stylesheet>