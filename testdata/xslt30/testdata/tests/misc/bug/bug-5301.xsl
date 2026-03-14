<?xml version="1.0" encoding="utf-8"?> 

<?spec xpath#id-general-comparisons?>
<!-- Saxon 6.5 can't parse this XPath expression -->                                                                                                                                                

<xsl:stylesheet
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<xsl:template match="a">
<o><xsl:if test="*/@z != */@z">There are duplicates</xsl:if></o>
</xsl:template>


</xsl:stylesheet>
