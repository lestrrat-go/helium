<xsl:transform
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 version="2.0"
>

<!-- Test "as" clauses on templates (with no schema)  -->
<?spec xslt#defining-templates?>
<?error XTTE0505?>
<!--XTTE0505: Required type of result of template match="/" is element(xhtml); supplied value
  has type element(html, xs:untyped)-->

<xsl:key name="authkey" match="ITEM" use="AUTHOR"/>
<xsl:key name="codekey" match="CATEGORY" use="@CODE"/>

<xsl:decimal-format name="comma" decimal-separator="," grouping-separator="."/>

<xsl:variable name="categories" select="//CATEGORY"/>

<xsl:variable name="now" select="current-dateTime()"/>

<xsl:param name="top-author">Bonner</xsl:param>

<xsl:template match="/" as="element(xhtml)">
  <xsl:apply-templates select="." mode="go"/>
</xsl:template>

<xsl:template match="/" mode="go" as="element(html)">
    <html>

    <xsl:comment>Generated at <xsl:value-of select="$now"/></xsl:comment>

    <xsl:call-template name="header">
        <xsl:with-param name="title" select="'Book List'"/>
    </xsl:call-template>
    
    <body leftmargin="100" >
    <xsl:apply-templates/>
    </body>
    </html>
</xsl:template>

<xsl:template name="header" as="element(head)">
    <xsl:param name="title" select="'Default Title'"/>
    <head>
      <xsl:choose>
      <xsl:when test="not($title)">
        <title><xsl:value-of select="$title"/></title>
      </xsl:when>
      <xsl:otherwise>
        <title>Untitled</title>
      </xsl:otherwise>
      </xsl:choose>
    </head>
</xsl:template>

<xsl:template match="BOOKLIST" as="element()*">

    <h2>This week's top author is <xsl:value-of select="$top-author"/></h2>
    <xsl:variable name="top-authors-books" select="key('authkey', $top-author)"/>
    
    <p>We stock the following <xsl:value-of select="count($top-authors-books)"/> books by this author:</p>
    <ul>
    <xsl:for-each select="$top-authors-books">
        <li><xsl:value-of select="TITLE"/></li>
    </xsl:for-each>
    </ul>
    
    <p>This author has written books in the following categories:</p>
    <ul>
    <xsl:for-each select="key('codekey', $top-authors-books/@CAT)/@DESC">
        <li><xsl:value-of select="."/></li>
    </xsl:for-each>
    </ul>

    <p>The average price of these books is: 
        <xsl:value-of select="format-number(
                                 sum($top-authors-books/PRICE)
                                   div count($top-authors-books),
                                     '$####.00')"/>
    </p>


    <h2>A complete list of books, grouped by author</h2>
    <xsl:apply-templates select="BOOKS" mode="by-author"/>

    <h2>A complete list of books, grouped by category</h2>
    <xsl:apply-templates select="BOOKS" mode="by-category"/>

    <h2>List of categories</h2>
    <xsl:apply-templates select="$categories">
        <xsl:sort select="@DESC" order="descending"/>
        <xsl:sort select="@CODE" order="descending"/>
    </xsl:apply-templates>

</xsl:template>   

<xsl:template match="BOOKS" mode="by-author" as="element(div)">
   <xsl:if test="apples = pears">
    <div>
    <xsl:for-each-group select="ITEM" group-by="AUTHOR">
    <xsl:sort select="AUTHOR" order="ascending"/>
    <xsl:sort select="TITLE" order="ascending"/>
    <h3>AUTHOR: <xsl:value-of select="AUTHOR"/></h3>
        <table>
        <xsl:for-each select="current-group()">
            <tr>
              <td width="100" valign="top"><xsl:number value="position()" format="i"/></td>
              <td>
                TITLE: <xsl:value-of select="TITLE"/><br/>
                CATEGORY: <xsl:value-of select="id(@CAT)/@DESC" />
                        (<xsl:value-of select="@CAT" />)
              </td>
            </tr>
        </xsl:for-each>
        </table>
        <hr/>
    </xsl:for-each-group>
    </div>
   </xsl:if> 
</xsl:template>

<xsl:template match="BOOKS" mode="by-category" as="element(div)">
    <div>
    <xsl:for-each-group select="ITEM" group-by="@CAT">
    <xsl:sort select="id(@CAT)/@DESC" order="ascending"/>
    <xsl:sort select="TITLE" order="ascending"/>
    <h3>CATEGORY: <xsl:value-of select="id(@CAT)/@DESC" /></h3>
        <ol>
        <xsl:for-each select="current-group()">
            <li>AUTHOR: <xsl:value-of select="AUTHOR"/><br/>
                TITLE: <xsl:value-of select="TITLE"/>
            </li>                
        </xsl:for-each>
        </ol>
        <hr/>
    </xsl:for-each-group>
    </div>
</xsl:template>

<xsl:template match="CATEGORY" as="element()*">
    <h4>CATEGORY <xsl:number value="position()" format="I"/></h4>
    <table>
    <xsl:for-each select="@*">
        <tr>
          <td><xsl:value-of select="name(.)"/></td>
          <td><xsl:value-of select="."/></td>
        </tr>
    </xsl:for-each>
    </table>
    <hr/>
</xsl:template>

</xsl:transform>	